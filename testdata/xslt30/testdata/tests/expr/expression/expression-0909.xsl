<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Test avg() over sequences -->

  <xsl:template match="/">
    <out>
      <a>
        <xsl:value-of select="avg(())"/>
      </a>
      <b>
        <xsl:value-of select="avg((1))"/>
      </b>
      <c>
        <xsl:value-of select="avg((1,2,4))"/>
      </c>
      <d>
        <xsl:value-of select="avg(//item/@val)"/>
      </d>
    </out>
  </xsl:template>

</xsl:stylesheet>
