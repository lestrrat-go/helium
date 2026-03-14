<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test tokenize() with case-blind matching. -->
  
  <xsl:template match="/">
    <xsl:param name="in">
      <xsl:text>Some unparsed &lt;BR&gt; HTML &lt;br&gt; text</xsl:text>
    </xsl:param>
    <out>
        <a><xsl:value-of select='tokenize($in, "\s*&lt;BR&gt;\s*", "i")' separator="|"/></a>
    </out>
  </xsl:template>

</xsl:stylesheet>