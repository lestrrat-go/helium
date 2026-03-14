<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="#all"
    version="3.0">
    
    <!-- xsl:number with large integers -->
    
    <xsl:template name="xsl:initial-template">
        <xsl:variable name="big" select="1234567890"/>
        <xsl:variable name="huge" select="$big * $big * $big"/>
        <out>
            <xsl:number value="(1 to 3)!($huge+.)" grouping-separator=":" grouping-size="9" start-at="3 2 1" format="[1(0)(&#x0660;)]"/>
        </out>
    </xsl:template>
    
</xsl:stylesheet>