<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- DOCUMENT: http://www.w3.org/TR/xslt -->
  <!-- SECTION: 7.6.2 Attribute Value Templates -->
  <!-- PURPOSE: Test of AVT containing XPath 2.0 comments. -->
  <xsl:template match="/">
    <out test="some{('exp', (: comments isn't }fun{ :) 23)}stuff"/>
  </xsl:template>
 
</xsl:stylesheet>
