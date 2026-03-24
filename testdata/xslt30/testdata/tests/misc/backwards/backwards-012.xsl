<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

  <!-- PURPOSE: effect of BC on xsl:sort -->

  <xsl:template name="main">
    <out>
      <xsl:perform-sort select="1 to 5">
        <xsl:sort select="(-., 'banana')"/>
      </xsl:perform-sort>
    </out>
  </xsl:template>

</xsl:stylesheet>
