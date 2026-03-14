<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test replace() with multi-digit substring references. -->
  <?spec fo#func-replace?>
  
  <xsl:template match="/">
    <xsl:param name="in">
      <xsl:text>The quick brown fox jumped over the lazy dog and immediately went into violent convulsions as a result of the dog being roused into an unusual state of hyperactivity.</xsl:text>
    </xsl:param>
    <out>
        <a><xsl:value-of select='replace($in, 
        "(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)(.*?\s)", 
        "$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,", "i")'/></a>
    </out>
  </xsl:template>

</xsl:stylesheet>