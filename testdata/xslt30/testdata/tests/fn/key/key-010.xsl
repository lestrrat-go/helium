<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:baz="http://xsl.lotus.com/ns1">

  <!-- Purpose: Test for xsl:key and key() with a qualified name. -->

  <xsl:key name="baz:mykey" match="div" use="title"/>

  <xsl:template match="doc">
    <root>
      <xsl:value-of select="key('baz:mykey', 'Introduction')/p"/>
      <xsl:text> </xsl:text>
      <xsl:value-of select="key('baz:mykey', 'Stylesheet Structure')/p"/>
      <xsl:text> </xsl:text>
      <xsl:value-of select="key('baz:mykey', 'Expressions')/p"/>
    </root>
  </xsl:template>

</xsl:stylesheet>
