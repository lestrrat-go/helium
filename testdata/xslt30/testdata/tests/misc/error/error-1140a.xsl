<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if the effective
                        value of the regex attribute of the
                           xsl:analyze-string instruction does not
                     conform to the required syntax for regular expressions, as
                     specified in . If the regular expression is
                     known statically (for example, if the attribute does not contain any expressions enclosed in curly brackets) then
                     the processor may signal the error as a static error. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE1140?>


  <xsl:template name="main">
      <out>
         <xsl:analyze-string select="'bananas'" regex="[A-Z">
            <xsl:matching-substring/>
         </xsl:analyze-string>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
