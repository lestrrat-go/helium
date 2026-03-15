<?xml version="1.0" encoding="UTF-8"?>
<!-- A dynamic call to the current() function is a dynamic error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="3.0">

<?error XTDE1360?>

  <xsl:variable name="f" select="current#0"/>
  <xsl:template name="main">
      <out>
        <xsl:for-each select="1 to 10">
            <xsl:sequence select="$f()"/>
        </xsl:for-each>  
      </out>
  </xsl:template>
 

</xsl:stylesheet>
