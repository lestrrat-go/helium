<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                  error if the value of the first argument to the
                     key function is not a valid QName,
                  or if there is no namespace declaration in scope for the prefix of the
                  QName, or if the name obtained by expanding the QName is not the same as the
                  expanded name of any xsl:key declaration in the 
                  containing package. If the processor is able to
                  detect the error statically (for example, when the argument is supplied as a
                  string literal), then the processor may optionally signal
                  this as a static error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE1260?>

   <xsl:key name="k" match="*" use="17"/>


  <xsl:template name="main">
      <out>
         <xsl:sequence select="key('my:k', 'abc')"/>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
