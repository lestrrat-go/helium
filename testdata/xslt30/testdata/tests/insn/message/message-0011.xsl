<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
   
   <xsl:variable name="data-1">
      <data1>
         <xsl:for-each select="1 to 30">
            <entry id="{.}" value="london-{.+.}"/>
         </xsl:for-each>
      </data1>     
   </xsl:variable>
   
   <xsl:variable name="data-2">
      <data2>
         <xsl:for-each select="1 to 30">
            <entry id="{.}" value="paris-{.+.}"/>
         </xsl:for-each>
      </data2>     
   </xsl:variable>
   
   <xsl:key name="one" match="entry" use="if (/data1) then key('two', @id, $data-1)/@id else @id"/>
   <xsl:key name="two" match="entry" use="if (/data2) then key('one', @id, $data-2)/@id else @id"/>
   
   <xsl:template name="main">
      <xsl:for-each select="1 to 20">
         <xsl:result-document href="out{.}.xml">
            <out>
               <xsl:message>
                  <message>
                     <message-start/>
                     <xsl:message select=" if (. mod 2 = 0) then $data-1 else $data-2"/>
                     <message-end/>
                  </message>
               </xsl:message>
            </out>
         </xsl:result-document>
      </xsl:for-each>
   </xsl:template>
   
</xsl:stylesheet>