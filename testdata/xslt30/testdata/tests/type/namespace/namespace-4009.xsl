<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:ped="http://www.test.com/">

  <!-- Purpose: Use xsl:element with namespace attribute and default reset; prefix known at stylesheet level. -->

  <xsl:template match="doc">
    <out>
      <xsl:element name="inner" namespace="http://www.test.com/" xmlns="">
        <yyy/>
      </xsl:element>
    </out>
  </xsl:template>

  <xsl:template match="text()"/>

</xsl:stylesheet>
