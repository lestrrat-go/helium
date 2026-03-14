<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <!-- Purpose: Test for resetting of an unspecified default namespace by copy-of. -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match = "/">
  <!-- create an RTF with no namespace nodes -->
  <xsl:variable name="x"><hello/></xsl:variable>
  <out>
    <xsl:element name="literalName" namespace="http://literalURI">
      <xsl:copy-of select="$x"/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>