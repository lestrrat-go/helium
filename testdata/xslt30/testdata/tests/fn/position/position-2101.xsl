<?spec fo#func-position?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">




 <xsl:template match="start">
  <xsl:param name="s" select="()"/>
  <xsl:apply-templates select="following-sibling::*[1]">
   <xsl:with-param name="s" select="(string(@name),$s)"/>
  </xsl:apply-templates>
 </xsl:template>


 <xsl:template match="end">
  <xsl:param name="s" select="()"/>
  <xsl:choose>
   <xsl:when test="empty($s)"/>
   <xsl:when test="$s[1]=@name">
    <xsl:apply-templates select="following-sibling::*[1]">
     <xsl:with-param name="s" select="$s[position()!=1]"/>
    </xsl:apply-templates>
   </xsl:when>
   <xsl:otherwise>
    <x length="{count($s)}" stack="{$s}" stack1="{$s[1]}" stack2="{$s[2]}" stack3="{$s[3]}"
     stack-2-="{$s[position()!=1]}"/>
    <xsl:apply-templates select=".">
     <xsl:with-param name="s" select="$s[position()!=1]"/>
    </xsl:apply-templates>
   </xsl:otherwise>
  </xsl:choose>
 </xsl:template>


 <xsl:template match="/">
  <xsl:variable name="x">
   <start name="b"/>
   <start name="c"/>
   <start name="d"/>
   <end name="b"/>
  </xsl:variable>


  <out>
   <xsl:apply-templates select="$x/*[1]"/>
  </out>


 </xsl:template>

</xsl:stylesheet>
