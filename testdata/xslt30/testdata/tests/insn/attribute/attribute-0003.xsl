<?xml version="1.0" encoding="UTF-8"?>

  <!-- FileName: atrs63 -->
    <!-- Purpose: Test that the value of the name attribute is cast to a string if supplied as a dynamic QName. -->

  
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" xmlns:p="http://ns.p.com/"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  
  <xsl:param name="n" select="xs:QName('p:local')"/>

  <xsl:template name="main">
  <out>
    <xsl:attribute name="{ $n }">content</xsl:attribute>
  </out>
  </xsl:template>


</xsl:stylesheet>