<?xml version="1.0" encoding="UTF-8"?>
<!--When a processor performs fallback for
                        an extension instruction
                        that is not recognized, if the instruction element has one or more
                           xsl:fallback children, then the content of each of the
                           xsl:fallback children must be
                        evaluated; it is a non-recoverable
                           dynamic error if it has no xsl:fallback
                        children.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="8.5">

  <!-- PURPOSE: Test no xsl:fallback for an unknown extension element. -->
  <?spec xslt#errors?><?error XPDE1450?>
  
  
  <xsl:template match="/">
      <out xmlns:burroughs="http://www.burroughs.com/"
           xmlns:honeywell="http://www.honeywell.com/"
           xmlns:cdc="http://www.cdc.com/"
           xsl:extension-element-prefixes="burroughs honeywell cdc">
        <cdc:non-existent-element processing="magic">
             <bananas/>
             <custard/>
             <pie>PIE!</pie>
        </cdc:non-existent-element>
      </out>
  </xsl:template>
   
</xsl:stylesheet>
