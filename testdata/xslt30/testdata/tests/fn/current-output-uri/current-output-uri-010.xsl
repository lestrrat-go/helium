<xsl:stylesheet 
    version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:f="urn:function" exclude-result-prefixes="xs f">
    
    <!-- current-output-uri() in xsl:merge-key  -->
    
    <xsl:output indent="yes" />

    <xsl:template name="xsl:initial-template" >
        <result>
            <xsl:merge>
                <xsl:merge-source for-each-item="0,10,20" select="for $i in . to . + 9 return $i * 2">
                    <xsl:merge-key select="., current-output-uri()"></xsl:merge-key>
                </xsl:merge-source>
                <xsl:merge-action>
                    <merged item="{.}">
                        <xsl:value-of select="substring-after(current-output-uri(), '/results/')" />
                    </merged>
                </xsl:merge-action>
            </xsl:merge>
        </result>
    </xsl:template>

</xsl:stylesheet>
