<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                  error if the value supplied as the
                     $property-name argument to the
                     system-property function is not a
                  valid QName, or if there is no namespace declaration in scope for the prefix
                  of the QName. If the processor is able to detect the error statically (for
                  example, when the argument is supplied as a string literal), then the
                  processor may optionally signal this as a static error. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE1390?>


  <xsl:template name="main">
    
      <out>        
          <xsl:sequence select="system-property('your:property')"/>        
      </out>
  </xsl:template>
  



</xsl:stylesheet>
