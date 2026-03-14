<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

  <!-- PURPOSE: effect of BC on unresolved function calls -->

  <xsl:template name="main">
    <out>
      <xsl:if test="2+2=5">
        <xsl:sequence select="unknown:function(42)" xmlns:unknown="http://www.unknown.com/"/>
      </xsl:if>
    </out>
  </xsl:template>

</xsl:stylesheet>
