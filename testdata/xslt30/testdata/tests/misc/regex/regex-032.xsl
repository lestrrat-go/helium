<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test matches() with multi-digit back-references. -->
  
  <xsl:template match="/">
    <xsl:param name="in">
      <xsl:text>The quick brown fox jumped over the lazy dog and immediately went into violent convulsions. The quick brown fox jumped over the lazy dog and immediately went into violent convulsions.</xsl:text>
    </xsl:param>
    <out>
        <a><xsl:value-of select='matches($in, 
        "(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s).*?\14.*?"
        )'/></a>
    </out>
  </xsl:template>

</xsl:stylesheet>