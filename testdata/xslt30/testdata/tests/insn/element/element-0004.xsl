<?xml version="1.0" encoding="UTF-8"?>

  <!-- FileName: element004 -->
    <!-- Purpose: Test that the value of the name attribute is cast to a string if supplied as a dynamic URI. -->

  
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" xmlns:p="http://ns.p.com/"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  
  <xsl:param name="ns" select="xs:anyURI('http://ns.p.com/2')"/>

  <xsl:template name="main">

    <xsl:element name="p:local" namespace="{ $ns }">content</xsl:element>

  </xsl:template>


</xsl:stylesheet>