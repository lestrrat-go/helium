<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if evaluating the select attribute or the
                     contained sequence constructor
                     of an xsl:namespace instruction results in a zero-length
                     string. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE0930?>

   <xsl:param name="p" select="''"/>

  <xsl:template name="main">
      <my:out>
        <xsl:namespace name="ns" select="$p"/>
      </my:out>
  </xsl:template>
  



</xsl:stylesheet>
