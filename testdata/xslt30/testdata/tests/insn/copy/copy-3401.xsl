<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- Purpose: Verify xml:lang attributes can be copied from source to result -->


  <xsl:template match="doc">
    <document>
      <xsl:copy-of select="@*"/>
    </document>
  </xsl:template>

</xsl:stylesheet>
