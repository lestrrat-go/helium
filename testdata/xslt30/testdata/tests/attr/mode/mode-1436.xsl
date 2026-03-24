<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- No match for document node so nothing should be output -->
    
    <xsl:mode streamable="no" on-no-match="deep-skip" />
    
    <xsl:template match="book | bktlong">
        <xsl:value-of select="name(.)" separator="|"/>
        <xsl:apply-templates />
    </xsl:template>
    
</xsl:stylesheet>
