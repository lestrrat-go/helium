<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#built-in-rule?>
	<!--Test for built-in template rule for attributes.-->

<xsl:template match="doc">
    <out>
       <xsl:apply-templates select="@test"/>
    </out>
</xsl:template>
 
</xsl:stylesheet>
