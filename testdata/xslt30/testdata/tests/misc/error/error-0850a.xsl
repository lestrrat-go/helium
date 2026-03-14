<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if the effective
                        value of the name attribute of an
                           xsl:attribute instruction is not a
                        lexical QName.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE0850?>

   <xsl:param name="p" select="'#'"/>

  <xsl:template name="main">
      <my:out>
        <xsl:attribute name="{concat('a', $p)}"/>
      </my:out>
  </xsl:template>
  



</xsl:stylesheet>
