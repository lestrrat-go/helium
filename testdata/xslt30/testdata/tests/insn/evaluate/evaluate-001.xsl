<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:my="http://www.example.com/my-functions" exclude-result-prefixes="my">

  <!-- PURPOSE test xsl:evaluate with a literal argument calling a stylesheet function -->


  <xsl:template name="main">
    <out>
      <xsl:evaluate xpath="'my:f()'"/>
    </out>
  </xsl:template>

  <xsl:function name="my:f" visibility="public">
    <xsl:sequence select="$p"/>
  </xsl:function>

  <xsl:param name="p" select="42"/>

</xsl:stylesheet>
