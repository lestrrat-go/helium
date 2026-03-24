<?spec fo#casting-to-string?>
<!-- BUG 5.3.1/007 -->
<!-- DESCRIPTION: When a number is converted to a string, XPath requires that "after the decimal point
    there must be as many, but only as many, more digits as are needed to uniquely
    distinguish the number from all other IEEE 754 numeric values". Saxon may display more
    digits than this. -->
    
<!-- NOTE: the rules have changed in XPath 2.0 -->

<bug-or-feature xsl:version="2.0"
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform"> 
  <xsl:value-of select="15.95 - 13.56"/>;
  <xsl:value-of select="1000000000000001"/>;
</bug-or-feature>

