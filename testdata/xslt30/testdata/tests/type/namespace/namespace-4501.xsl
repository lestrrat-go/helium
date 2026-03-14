<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test for resetting of an unspecified default namespace by copy-of. -->
  <!-- extended variant of namespace137 to check special cases -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match = "/">
  <!-- create an RTF with a mix of namespaced and non-namespaced elements -->
  <xsl:variable name="x">
    <xsl:element name="hello1"/>
    <xsl:element name="hello2" namespace="http://literalURI">
      <xsl:element name="hiya" namespace=""/>
    </xsl:element>
    <xsl:text>&#10;</xsl:text>
    <xsl:element name="hello3" namespace="http://literalURI2">
      <xsl:element name="yo1" namespace="http://literalURI"/>
      <xsl:element name="yo2" namespace=""/>
    </xsl:element>
    <xsl:text>&#10;</xsl:text>
    <xsl:element name="hello4">
      <xsl:element name="hey" namespace=""/>
    </xsl:element>
  </xsl:variable>
  <!-- Now start an output tree, with a namespace node, and copy in the RTF -->
  <out>
    <xsl:text>&#10;</xsl:text>
    <xsl:element name="literalName" namespace="http://literalURI">
      <xsl:text>&#10;</xsl:text>
      <xsl:copy-of select="$x"/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>
