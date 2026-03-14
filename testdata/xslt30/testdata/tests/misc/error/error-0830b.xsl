<?xml version="1.0" encoding="UTF-8"?>
<!--In the case of an xsl:element instruction with no
                        namespace attribute, it is a non-recoverable dynamic error if the
                        effective value of the
                        name attribute is a lexical QName
                     whose prefix is not declared in an in-scope namespace declaration for the
                        xsl:element instruction.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE0830?>



  <xsl:template name="main">
      <my:out>
        <xsl:element name="{concat('your:element', '10')}"/>
      </my:out>
  </xsl:template>
  



</xsl:stylesheet>
