<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                              error if the effective
                              value of an attribute written using curly brackets, in a
                           position where an attribute
                              value template is permitted, is a value that is not one of
                           the permitted values for that attribute. If the processor is able to
                           detect the error statically (for example, when any XPath expressions
                           within the curly brackets can be evaluated statically), then the
                           processor may optionally signal this as a static error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?error XTDE0030?>

   <xsl:param name="x" select="'bananas'"/>

  <xsl:template name="main">
      <out>
         <xsl:message terminate="{$x}"/>
      </out>
  </xsl:template>

</xsl:stylesheet>
