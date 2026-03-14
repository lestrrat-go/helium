<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
    <!-- Purpose: The name attribute of xsl:element is interpreted as an 
       attribute value template. -->

<xsl:template match="/">
  <out>
    <xsl:element name="{.//doc2/doc3/a/@level}"/>
    <xsl:element name="{.//*[starts-with(name(.),'ba')]}"/>
  </out>
</xsl:template>

</xsl:stylesheet>
