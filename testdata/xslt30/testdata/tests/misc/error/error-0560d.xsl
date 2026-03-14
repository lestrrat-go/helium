<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if xsl:apply-imports or
                        xsl:next-match is evaluated when the current template rule is 
                     absent.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                version="2.0">

<?error XTDE0560?>



  <xsl:template name="main">
      <out>
        <xsl:next-match/>
      </out>
  </xsl:template>
  


</xsl:stylesheet>
