<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if the effective
                        value of the name attribute of the
                           xsl:processing-instruction instruction is
                     not both an NCName and a PITarget.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE0890?>

   <xsl:param name="p" select="'ns'"/>

  <xsl:template name="main">
      <my:out>
        <xsl:processing-instruction name="xml">version='1.0'</xsl:processing-instruction>
      </my:out>
  </xsl:template>
  



</xsl:stylesheet>
