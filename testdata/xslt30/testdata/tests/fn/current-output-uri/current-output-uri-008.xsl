<xsl:stylesheet 
    version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:f="urn:function" exclude-result-prefixes="xs f">
    
    <!-- current-output-uri() initial named template, different with/without temp output state -->
    
    <xsl:output indent="yes" />
    
    <xsl:variable name="base" as="element()">
        <base>
            <value>
                <xsl:value-of select="current-output-uri()" />
            </value>
        </base>
    </xsl:variable>
    
    <xsl:function name="f:start">
        <xsl:apply-templates select="$base" />
    </xsl:function>
    
    <xsl:template match="$base">
        <output-uri value="{base/text()}" from="variable" />
        <output-uri value="{current-output-uri()}" from="template-rule" />
        <xsl:apply-templates select="$base/value"/>
    </xsl:template>
    
    <xsl:template match="$base/value[current-output-uri()]" >
        <output-uri wrong="should be empty in patterns" /> 
    </xsl:template>

    <xsl:template match="$base/value[empty(current-output-uri())]" >
        <output-uri correct="output-uri is empty in patterns" /> 
    </xsl:template>

    <xsl:template name="start" >
        <result>
            <output-uri value="{current-output-uri()}" from="initial-template"/>
            <xsl:apply-templates select="$base" />
            <in-function>
                <xsl:copy-of select="f:start()" />
            </in-function>
            <in-pattern>
                <xsl:for-each-group select="-1 to +1" group-starting-with=".[.=count(current-output-uri())]">
                    <group at="{.}"/>
                </xsl:for-each-group>
            </in-pattern>
        </result>
    </xsl:template>

</xsl:stylesheet>
