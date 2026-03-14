<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">

                 <!--It is a type error if some item selected
                     by a particular merge key in one input sequence is not comparable using the
                     XPath le operator with some item selected by the corresponding
                     sort key in another input sequence.-->
                     
   <xsl:template name="main">
      <out>
         <xsl:merge>
           <xsl:merge-source name="a" select="1 to 50">
             <xsl:merge-key select="."/>
           </xsl:merge-source>
           <xsl:merge-source name="b" select="1 to 50">
             <xsl:merge-key select="xs:dayTimeDuration('P1D')*."/>
           </xsl:merge-source>
           <xsl:merge-action>
             22
           </xsl:merge-action>
         </xsl:merge>
      </out>
   </xsl:template>
</xsl:stylesheet>
