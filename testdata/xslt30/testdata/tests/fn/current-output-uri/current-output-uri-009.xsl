<xsl:stylesheet 
    version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:f="urn:function" exclude-result-prefixes="xs f">
    
    <!-- current-output-uri() in xsl:sort  -->
    
    <xsl:output indent="yes" />

    <xsl:template name="xsl:initial-template" >
        <xsl:apply-templates select="4,3,8,1,8">
            <!-- if current-output-uri() doesn't return empty-sequence(), this would throw XTTE1020 -->
            <xsl:sort select="., current-output-uri()" />
        </xsl:apply-templates>
    </xsl:template>

</xsl:stylesheet>
