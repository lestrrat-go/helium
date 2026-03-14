<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" xmlns:f="http://f.com/">

<?spec xslt#patterns?>

    <!-- Purpose: Test union operator not at the top level of a pattern. Allowed in 3.0 -->
  
<xsl:key name="k" match="*" use="@att1"/>  

<xsl:template match="key('k', 'c')//(bar|baz)" priority="5">
 <out><xsl:value-of select="@att1"/></out>
</xsl:template>

<xsl:template match="/">
  <xsl:apply-templates select=".//*"/>
</xsl:template>

<xsl:template match="*"/>

</xsl:stylesheet>
