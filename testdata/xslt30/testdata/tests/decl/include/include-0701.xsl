<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:include href="include-0701b.xsl"/>

<?spec xslt#include?>
<!-- Test for xsl:include.  -->

<xsl:template match="/*">
  <out>
     <xsl:apply-templates/>
  </out>
</xsl:template>
 
<xsl:template match="title">
  Includ01-title: 
  <xsl:apply-templates/>
</xsl:template>

<xsl:include href="include-0701c.xsl"/>

<xsl:template match="chapter">
  <xsl:value-of select="@num"/> .  
  <xsl:value-of select="."/>
</xsl:template>

</xsl:stylesheet>
