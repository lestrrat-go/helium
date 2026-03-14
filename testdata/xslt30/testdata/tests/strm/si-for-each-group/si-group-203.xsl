<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:param name="input-uri" select="'si-group-203.xml'"/>
    
    <xsl:strip-space elements="*"/>
    
    <xsl:template name="main">
        <xsl:source-document streamable="yes" href="{$input-uri}">
            <Root>
                <xsl:for-each-group select="//Item/copy-of()" group-starting-with="Item[processing-instruction('start')]">
                    <xsl:for-each-group select="current-group() except ." group-ending-with="Item[processing-instruction('end')]">
                        <Group>
                            <xsl:copy-of select="current-group()[position() != last()]"/>
                        </Group>
                    </xsl:for-each-group>
                </xsl:for-each-group>
            </Root>
        </xsl:source-document>
    </xsl:template>
    
</xsl:stylesheet>