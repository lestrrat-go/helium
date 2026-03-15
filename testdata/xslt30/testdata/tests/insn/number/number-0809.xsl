<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: test ordinal numbering e.g. 1st, 2nd, 3rd -->
  <xsl:template match="doc">
    <out>
        1 = <xsl:number value="1" format="1" ordinal="yes" />;
        2 = <xsl:number value="2" format="1" ordinal="yes" />;            
        3 = <xsl:number value="3" format="1" ordinal="yes" />;
        4 = <xsl:number value="4" format="1" ordinal="yes" />;        
        10 = <xsl:number value="10" format="1" ordinal="yes" />;
        10 = <xsl:number value="11" format="1" ordinal="yes" />;        
        13 = <xsl:number value="13" format="1" ordinal="yes" />;
        20 = <xsl:number value="20" format="1" ordinal="yes"  />;
        21 = <xsl:number value="21" format="1" ordinal="yes"  />; 
        78 = <xsl:number value="78" format="1" ordinal="yes"  />;               
        100 = <xsl:number value="100" format="1" ordinal="yes" />;
        115 = <xsl:number value="115" format="1" ordinal="yes" />;
        134 = <xsl:number value="134" format="1" ordinal="yes" />;
        200 = <xsl:number value="200" format="1" ordinal="yes" />;
        201 = <xsl:number value="201" format="1" ordinal="yes" />;
        210 = <xsl:number value="210" format="1" ordinal="yes" />;
        230 = <xsl:number value="230" format="1" ordinal="yes"  />;
        1000 = <xsl:number value="1000" format="1" ordinal="yes" />;
        1005 = <xsl:number value="1005" format="1" ordinal="yes" />;
        1998 = <xsl:number value="1998" format="1" ordinal="yes" />;  
        2000 = <xsl:number value="2000" format="1" ordinal="yes" />;
        2002 = <xsl:number value="2002" format="1" ordinal="yes" />;                      
        2134816 = <xsl:number value="2134816" format="1" ordinal="yes" />;       
    </out>
  </xsl:template>
 
</xsl:stylesheet>
