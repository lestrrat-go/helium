<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:z="http://example.com/z">
    
    <!-- Purpose: Test suppress-indentation -->


    <xsl:template match="/">
       <xsl:result-document method="xml" indent="yes" suppress-indentation="p z:p">
        <book>
          <chapter>
             <xsl:for-each select="1 to 5">
               <p>In the beginning <wizard><cap>G</cap><small>andalf</small></wizard> created the Shire.</p>
             </xsl:for-each>  
             <xsl:for-each select="1 to 5">
               <z:p>In the end <hobbit><big>B</big><little>ilbo</little></hobbit> found the Ring.</z:p>
             </xsl:for-each>
           </chapter>
         </book>
        </xsl:result-document>       
    </xsl:template>
</xsl:transform>
