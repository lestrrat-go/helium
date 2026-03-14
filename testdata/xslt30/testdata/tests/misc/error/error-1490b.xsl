<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error for a transformation to generate two or more final result trees with the same
                     URI.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE1490?>


  <xsl:template name="main">
    
      <out> 
          <xsl:result-document href="temp.out">
               <apple/>
          </xsl:result-document>   
          <xsl:result-document href="temp.out">
               <pear/>
          </xsl:result-document>             
      </out>
  </xsl:template>
  




</xsl:stylesheet>
