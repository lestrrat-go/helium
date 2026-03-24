<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test proper construction of an RTF containing multiple top level nodes. -->

<xsl:variable name="foo">
  <b>First element</b>
  Second element
  <FORM METHOD="post">
    <input type="length" size="30" />
  </FORM>
</xsl:variable>

<xsl:template match="/">
  <out><xsl:copy-of select="$foo"/></out>
</xsl:template>

</xsl:stylesheet>