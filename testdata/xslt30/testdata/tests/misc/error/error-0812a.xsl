<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if a value
                        other than #default is specified for either the
                           stylesheet-prefix or the result-prefix
                        attributes of the xsl:namespace-alias element when there is
                        no in-scope binding for that namespace prefix.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0812?>



  <xsl:template name="main">
      <my:out>
        <xsl:sequence select="2"/>
      </my:out>
  </xsl:template>
  
   <xsl:namespace-alias stylesheet-prefix="my" result-prefix="xs"/>
   <xsl:namespace-alias stylesheet-prefix="your" result-prefix="xml"/>


</xsl:stylesheet>
