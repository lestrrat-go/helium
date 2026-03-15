<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!-- It is a dynamic error if any input
     sequence to an xsl:merge instruction contains two items that
     are not correctly sorted according to the merge key values defined on the
     xsl:merge-key children of the corresponding
     xsl:merge-source element, when compared using the collation
     rules defined by the attributes of the corresponding
     xsl:merge-key children of the xsl:merge
     instruction, unless the attribute sort-before-merge is present
     with the value yes.-->
     
   <xsl:template name="main">
      <out>
         <xsl:merge>
           <xsl:merge-source name="a" select="1 to 50">
             <xsl:merge-key select="."/>
           </xsl:merge-source>
           <xsl:merge-source name="b" select="reverse(1 to 50)">
             <xsl:merge-key select="."/>
           </xsl:merge-source>
           <xsl:merge-action expand-text="yes">
             {current-merge-group()}
           </xsl:merge-action>
         </xsl:merge>
      </out>
   </xsl:template>
</xsl:stylesheet>
