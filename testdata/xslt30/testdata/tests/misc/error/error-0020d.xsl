<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an
                           attribute (other than an attribute written using curly brackets in a
                           position where an attribute
                              value template is permitted) contains a value that is not
                           one of the permitted values for that attribute.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:key - bad key name -->
<?spec xslt#errors?><?error XTSE0020?>


  <xsl:key name="foo::bar" match="doc" use="17"/>

  <xsl:template match="doc">
      <out>
         <xsl:copy/>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>

</xsl:stylesheet>
