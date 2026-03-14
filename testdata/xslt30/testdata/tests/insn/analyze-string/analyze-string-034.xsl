<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
 xmlns:f="http://localhost/" exclude-result-prefixes="f">

  <!-- PURPOSE: test regex-group is empty inside a stylesheet function. -->
  
  <xsl:template match="/">
    <out>
      <xsl:analyze-string select="doc" regex="(M.*?y )|(h.*?d )|(l.*?e )" flags="s">
       <xsl:matching-substring>[<xsl:value-of select="f:regex-group(0)"/>(<xsl:value-of select="f:regex-group(2)"/>)]</xsl:matching-substring>
       <xsl:non-matching-substring><xsl:value-of select="."/></xsl:non-matching-substring>
      </xsl:analyze-string>      
    </out>
  </xsl:template>
  
  <xsl:function name="f:regex-group">
    <xsl:param name="n"/>
    <xsl:sequence select="regex-group($n)"/>
  </xsl:function>

</xsl:stylesheet>