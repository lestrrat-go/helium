<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an
                           attribute (other than an attribute written using curly brackets in a
                           position where an attribute
                              value template is permitted) contains a value that is not
                           one of the permitted values for that attribute.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: Cannot specify tunnel="yes on a stylesheet parameter -->
<?spec xpath#errors?><?error XTSE0020?>

  <xsl:template match="doc">
      <out/>
  </xsl:template>
  
  <xsl:param name="p" tunnel="yes"/>

</xsl:stylesheet>
