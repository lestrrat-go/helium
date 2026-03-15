<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!--It is a static error if for any named or unnamed mode, a package explicitly specifies two 
    conflicting values for the same attribute in different xsl:mode declarations having the same 
    import precedence, unless there is another definition of the same attribute with higher import 
    precedence. The attributes in question are the attributes other than name on the xsl:mode element-->
    
   <xsl:mode name="m" on-multiple-match="fail"/>
   <xsl:mode name="m" on-multiple-match="use-last"/>
    
   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
</xsl:stylesheet>
