<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the
                        xsl:analyze-string instruction contains neither an
                        xsl:matching-substring nor an
                        xsl:non-matching-substring element.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1130?>


  <xsl:template name="main">
      <xsl:variable name="pop">
         <a/>
         <b/>
         <c/>
      </xsl:variable>
      <out>
         <xsl:analyze-string select="'bananas'" regex="[A-Z]+"/>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
