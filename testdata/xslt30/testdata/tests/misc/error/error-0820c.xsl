<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if the effective
                        value of the name attribute of the
                           xsl:element instruction is not a lexical QName. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE0820?>



  <xsl:template name="main">
      <my:out>
        <xsl:element name="{concat('my:element', '#')}"/>
      </my:out>
  </xsl:template>
  



</xsl:stylesheet>
