<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if a named or
                     unnamed decimal format contains two
                     conflicting values for the same attribute in different
                        xsl:decimal-format declarations having the same import precedence, unless there is
                     another definition of the same attribute with higher import precedence.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1290?>

   <xsl:decimal-format name="d" percent="%"/>
   <xsl:decimal-format name="d" percent="*"/>





  <xsl:template name="main">
      <xsl:variable name="e" as="element()">
         <e/>
      </xsl:variable>
      <xsl:variable name="d">
         <e/>
      </xsl:variable>    
      <out>
        <xsl:sequence select="format-number(12, '##0', 'd')"/>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
