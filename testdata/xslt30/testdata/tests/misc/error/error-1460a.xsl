<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if the effective
                        value of the format attribute of an
                           xsl:result-document element is not a valid
                        EQName, or if it does not
                     match the expanded QName of an
                        output definition in the
                        stylesheet. If the processor is able
                     to detect the error statically (for example, when the format
                     attribute contains no curly brackets), then the processor
                        may optionally signal this as a static error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test xsl:result-document with unknown format -->
<?spec xslt#errors?><?error XTSE1460?>

   <xsl:param name="i">yes</xsl:param>
   <xsl:output name="sec" method="xml" indent="yes" omit-xml-declaration="yes"/>
   <xsl:template match="/">
      <out>
         <xsl:result-document format="sec-wrong-format" href="error234output.xml">
            <data value="3">bananas</data>
         </xsl:result-document>
         <xsl:copy-of select="document('file:/c:/x31temp.xml')"/>
      </out>
   </xsl:template>
 
</xsl:stylesheet>
