<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
   version="2.0"
>

<!-- PURPOSE: Test generate-id() on namespace nodes -->
<!-- modified 2002-08-08 to give reproducable results. The test now simply checks that
     all the namespace nodes have distinct id values -->
<!-- modified 2002-08-08 to give even more reproducable results. Output is now
     well-formed, and sorted.-->


<xsl:template match="/">
 <out>
 <xsl:for-each-group select="//*/namespace::*" group-by="generate-id()">
   <xsl:sort select="name()"/>
   <xsl:sort select="."/>
   <namespace name="{name()}" value="{.}" count="{count(current-group())}"/>
 </xsl:for-each-group>
 </out>
</xsl:template>
 
</xsl:stylesheet>
