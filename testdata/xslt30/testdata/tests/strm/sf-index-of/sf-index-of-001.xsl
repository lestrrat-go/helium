<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://local.functions/"
    exclude-result-prefixes="map xs f">
    
    <xsl:variable name="RUN" static="yes" select="true()"/>
    
    <!-- Test of xsl:source-document calling index-of(), first argument consuming and climbing  -->
    
    <xsl:template name="i-001" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="index-of(/account/transaction/@date, '2007-01-21')"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document calling index-of(), first argument motionless, second argument streamed  -->
    
    <xsl:template name="i-002" use-when="$RUN">
      <xsl:variable name="accounts" as="element()*">
      	<account-number>01234567a</account-number>
      	<account-number>01234567b</account-number>
      	<account-number>01234567c</account-number>
      	<account-number>01234567d</account-number>
      	<account-number>01234567</account-number>
      	<account-number>01234567e</account-number>
      	<account-number>01234567f</account-number>
      </xsl:variable>	
      
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="index-of($accounts, /account/account-number)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document calling index-of(), empty case  -->
    
    <xsl:template name="i-003" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="index-of(/account/transactionDATA/@date, '2007-01-21')"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming index-of(): collation argument present -->
    
    <xsl:template name="i-050" use-when="$RUN">
      <xsl:param name="c" select="'http://www.w3.org/2005/xpath-functions/collation/codepoint'"/>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="index-of(outermost(//AUTHOR)/string(.), 'Jane Austen', $c)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming index-of(): collation argument present, unknown collation -->
    
    <xsl:template name="i-051" use-when="$RUN">
      <xsl:param name="c" select="'http://www.w3.org/2005/xpath-functions/collation/codepoint/unknown'"/>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="index-of(outermost(//AUTHOR)/string(.), 'Jane Austen', $c)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming index-of(): collation argument present, unknown collation, recovery case -->
    
    <xsl:template name="i-052" use-when="$RUN">
      <xsl:param name="c" select="'http://www.w3.org/2005/xpath-functions/collation/codepoint/unknown'"/>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try>
            <xsl:value-of select="index-of(outermost(//AUTHOR)/string(.), 'Jane Austen', $c)"/>
            <xsl:catch errors="*:FOCH0002" select="'caught'"/>
          </xsl:try>  
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming index-of(): collation argument obtained from streamed input -->
    
    <xsl:template name="i-053" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <out>
          <xsl:value-of select="index-of(('a', 'b', 'c'), 'a', special/codepointCollation)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming index-of(): collation argument obtained from streamed input, unknown collation -->
    
    <xsl:template name="i-054" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <out>
          <xsl:value-of select="index-of(('a', 'b', 'c'), 'a', special/unknownCollation)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming index-of(): collation argument obtained from streamed input, unknown collation, recovery case -->
    
    <xsl:template name="i-055" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <out>
          <xsl:try>
            <xsl:value-of select="index-of(('a', 'b', 'c'), 'a', special/unknownCollation)"/>
            <xsl:catch errors="*:FOCH0002" select="'caught'"/>
          </xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
                                   
    
    
    
</xsl:stylesheet>