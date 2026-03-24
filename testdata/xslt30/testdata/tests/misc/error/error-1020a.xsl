<?xml version="1.0" encoding="UTF-8"?>
<!--If any sort key value, after
                           atomization and any type
                        conversion required by the data-type
                        attribute, is a sequence containing more than one item, then the effect
                        depends on whether the xsl:sort element is processed with XSLT 1.0
                              behavior. With XSLT 1.0 behavior, the effective
                        sort key value is the first item in the sequence. In other cases, this is a
                           type error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTTE1020?>


  <xsl:template name="main">
      <out>
         <xsl:for-each select="1 to 5">
            <xsl:sort select="(.,.,.)"/>
            <xsl:value-of select="."/>
         </xsl:for-each>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
