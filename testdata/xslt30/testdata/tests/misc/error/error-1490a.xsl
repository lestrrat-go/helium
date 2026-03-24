<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error for a transformation to generate two or more final result trees with the same
                     URI.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test xsl:result-document writing twice to same destination -->
<?spec xslt#errors?><?error XTDE1490?>

   <xsl:param name="i">yes</xsl:param>
   <xsl:output name="sec" method="xml" indent="yes" omit-xml-declaration="yes"/>
   <xsl:template match="/">
      <out>
         <xsl:result-document format="sec" href="error249output.xml">
            <data value="3">bananas</data>
         </xsl:result-document>
         <xsl:result-document format="sec" href="error249output.xml">
            <data value="3">bananas</data>
         </xsl:result-document>    
      </out>
   </xsl:template>
 
</xsl:stylesheet>
