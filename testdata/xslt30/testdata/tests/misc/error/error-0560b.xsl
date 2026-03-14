<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if xsl:apply-imports or
                        xsl:next-match is evaluated when the current template rule is 
                     absent.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: Calling xsl:apply-imports when evaluating a global variable -->
<?spec xslt#errors?><?error XTDE0560?>

  <xsl:template match="/">
      <out>
         <xsl:value-of select="$g"/>
      </out>
  </xsl:template>
  
  <xsl:variable name="g">
      <a>
         <xsl:apply-imports/>
      </a>
  </xsl:variable>

</xsl:stylesheet>
