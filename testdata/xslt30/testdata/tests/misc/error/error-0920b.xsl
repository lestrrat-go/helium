<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if the effective
                        value of the name attribute of the
                           xsl:namespace instruction is neither a
                     zero-length string nor an NCName, or if
                     it is xmlns. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE0920?>

   <xsl:param name="p" select="'23'"/>

  <xsl:template name="main">
      <my:out>
        <xsl:namespace name="{$p}">http://my.com/</xsl:namespace>
      </my:out>
  </xsl:template>
  



</xsl:stylesheet>
