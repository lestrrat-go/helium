<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns="http://testguys.com/" xmlns:ped="http://www.test.com/">

  <!-- Purpose: Use namespace attribute, but namespace new at that point; default was set. -->
  <!-- Requested name has no prefix, and we can get by without it. -->

  <xsl:template match="doc">
    <out>
      <xsl:element name="inner" namespace="http://other.com/">
        <yyy/>
      </xsl:element>
    </out>
  </xsl:template>

  <xsl:template match="text()"/>

</xsl:stylesheet>
