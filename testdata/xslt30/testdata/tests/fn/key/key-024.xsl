<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: idkey29 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.2 -->
    <!-- Purpose: Use content of an element as the "use" value. -->

<xsl:key name="mykey" match="div" use="text()"/>

<xsl:template match="doc">
 <out>
  <xsl:value-of select="key('mykey','Intro Section')/@title"/><xsl:text>,</xsl:text>
  <xsl:value-of select="key('mykey','SS Section')/@title"/><xsl:text>,</xsl:text>
  <xsl:value-of select="key('mykey','Exp Section')/@title"/><xsl:text>,</xsl:text>
  <xsl:value-of select="key('mykey',' ')/@title"/><xsl:text>,</xsl:text>
  <xsl:value-of select="key('mykey','Sort Section')/@title"/>
 </out>
</xsl:template>

</xsl:stylesheet>
