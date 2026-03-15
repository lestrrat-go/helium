<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
				exclude-result-prefixes="xsl" version="1.0">

<!--Test for Multiple Input documents.-->

<xsl:template match="doc">
    <out>
       <xsl:apply-templates select="@test"/>
    </out>
</xsl:template>
 
</xsl:stylesheet>
