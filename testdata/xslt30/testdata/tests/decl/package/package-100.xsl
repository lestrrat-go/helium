<?xml version="1.0" encoding="UTF-8"?>
<xsl:package
    name="http://example.com/csv-parser"
    package-version="1.0.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:csv="http://example.com/csv"
    exclude-result-prefixes="xs csv"
    declared-modes="yes"
    version="3.0">
    
    <xsl:mode name="csv:parse-line" visibility="public"/>
    
    <xsl:mode name="csv:parse-field" on-no-match="shallow-copy" visibility="public"/>
    
    <xsl:mode name="csv:post-process" on-no-match="shallow-copy" visibility="public"/>
    
    <xsl:variable name="csv:line-separator" as="xs:string" select="'\r\n?|\n\r?'" visibility="public"/>
    
    <xsl:variable name="csv:field-separator" as="xs:string" select="','" visibility="public"/>
    
    <xsl:variable name="csv:quote" as="xs:string" select="'&quot;'" visibility="public"/>
    
    <xsl:variable name="csv:validated-quote" as="xs:string" visibility="private" select="
            if (string-length($csv:quote) ne 1) 
            then error(xs:QName('csv:ERR001'), 'Incorrect length for $csv:quote, should be 1') 
            else $csv:quote" />
    
    <xsl:template match="." mode="csv:parse-line">
        <row>
            <xsl:apply-templates 
                select="tokenize(., $csv:field-separator)" 
                mode="csv:parse-field" />
        </row>
    </xsl:template>
    
    <xsl:template match="." mode="csv:parse-field">
        <xsl:variable name="string-body-pattern" as="xs:string" select="'([^' || $csv:validated-quote || ']*)'"/>
        <xsl:variable name="quoted-value" as="xs:string" select="$csv:validated-quote || $string-body-pattern || $csv:validated-quote"/>
        <xsl:variable name="unquoted-value" as="xs:string" select="'(.+)'"/>
        
        <field xsl:use-attribute-sets="csv:field-attributes" xsl:expand-text="yes">{
            csv:preprocess-field(
              replace(., 
                      $quoted-value || '|' || $unquoted-value, 
                      '$1$2'))
        }</field>
    </xsl:template>
    
    
    <xsl:attribute-set name="csv:field-attributes" visibility="public">
        <xsl:attribute name="quoted" 
            select="if (starts-with(., $csv:quote)) 
                    then 'yes' 
                    else 'no'" />        
    </xsl:attribute-set>

    <xsl:function name="csv:preprocess-line" as="xs:string?" visibility="public">
        <xsl:param name="line" as="xs:string" />
        <xsl:sequence select="normalize-space($line)" />
    </xsl:function>
    
    <xsl:function name="csv:parse" visibility="final">
        <xsl:param name="input" as="xs:string" />   
        <xsl:variable name="result" as="element()">
            <csv>
                <xsl:apply-templates 
                    select="(tokenize($input, $csv:line-separator) 
                            ! csv:preprocess-line(.))" 
                    mode="csv:parse-line" />
            </csv>
        </xsl:variable>
        <xsl:apply-templates select="$result" mode="csv:post-process" />
    </xsl:function>
    
    <xsl:function name="csv:preprocess-field" as="xs:string">
        <xsl:param name="field" as="xs:string" />
        <xsl:sequence select="$field" />
    </xsl:function>
    
    <!--* Templates ... *-->

</xsl:package>