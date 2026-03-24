<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                  error if the argument passed to the
                     function-available function does
                  not evaluate to a string that is a valid EQName, or if the value is a lexical QName
                  with a prefix for which no namespace declaration is present in the static context. If the processor
                  is able to detect the error statically (for example, when the argument is
                  supplied as a string literal), then the processor may
                  optionally signal this as a static
                     error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE1400?>

  <xsl:template name="main">
    
      <out>        
          <xsl:sequence select="function-available('1234')"/>        
      </out>
  </xsl:template>
  



</xsl:stylesheet>
