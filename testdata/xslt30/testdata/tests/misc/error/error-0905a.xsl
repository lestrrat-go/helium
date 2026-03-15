<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                     error if the string value of the new namespace
                     node is not valid in the lexical space of the datatype xs:anyURI,
                     or if it is the string http://www.w3.org/2000/xmlns/.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE0905?>

   <xsl:param name="p" select="'ns'"/>

  <xsl:template name="main">
      <my:out>
        <xsl:namespace name="ns">####</xsl:namespace>
      </my:out>
  </xsl:template>
  



</xsl:stylesheet>
