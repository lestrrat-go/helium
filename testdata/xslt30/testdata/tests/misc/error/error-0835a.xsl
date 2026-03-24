<?xml version="1.0" encoding="UTF-8"?>
<!-- It is a non-recoverable dynamic
                        error if the effective
                        value of the namespace attribute of the
                           xsl:element instruction is not in the
                     lexical space of the xs:anyURI datatype or if it is the string
                        http://www.w3.org/2000/xmlns/.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE0835?>



  <xsl:template name="main">
      <my:out>
        <xsl:element name="your:element" namespace="##not-a-uri"/>
      </my:out>
  </xsl:template>
  



</xsl:stylesheet>
