<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <xsl:template match="p">
    <wrap>
      <xsl:next-match/>
    </wrap>
  </xsl:template>

</xsl:stylesheet>
